<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    xmlns:map="http://www.w3.org/2005/xpath-functions/map" 
    exclude-result-prefixes="xs math map" 
    expand-text="true" 
    version="3.0">
    
    <xsl:param name="STREAMABLE" as="xs:boolean" static="yes" select="true()"/>
    
    <xsl:mode _streamable="{$STREAMABLE}" on-no-match="shallow-skip" use-accumulators="item1-count subitem groups"/>
    
    <xsl:accumulator name="item1-count" as="xs:integer" initial-value="0" _streamable="{$STREAMABLE}">
        <xsl:accumulator-rule match="TRANSACTION" select="0"/>
        <xsl:accumulator-rule match="TRANSACTION/ITEM1" select="$value + 1"/>
    </xsl:accumulator>
    
    <xsl:accumulator name="subitem" as="xs:integer" initial-value="0" _streamable="{$STREAMABLE}">
        <xsl:accumulator-rule match="TRANSACTION/ITEM2/SUBITEM2.1/text()" select="xs:integer(.)"/>
    </xsl:accumulator>
    
    <xsl:accumulator name="groups" as="map(xs:string, xs:integer*)" initial-value="map{}" _streamable="{$STREAMABLE}">
        <xsl:accumulator-rule match="TRANSACTION/ITEM2/SUBITEM2.2/GROUPING-KEY/text()" 
            select="let $key := string(),
            $count := accumulator-before('item1-count'),
            $sum := accumulator-before('subitem')
            return if (not(map:contains($value, $key)))
            then map:put($value, $key, ($count, $sum))
            else let $value-seq := $value($key)
            return map:put($value, $key, ($count + $value-seq[1], $sum + $value-seq[2]))"/>
    </xsl:accumulator>
    
    <xsl:template match="TRANSACTION-LIST">
        <xsl:copy>
            <xsl:apply-templates/>
            <xsl:variable name="groups" select="accumulator-after('groups')"/>
            <xsl:for-each select="map:keys($groups)">
                <transaction key="{.}">
                    <count>{$groups(.)[1]}</count>
                    <amount>{$groups(.)[2]}</amount>
                </transaction>
            </xsl:for-each>
        </xsl:copy>
    </xsl:template>
    
</xsl:stylesheet>