<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- stylesheet to extract selected parts from a MusicXML score -->
    
    <xsl:param name="in" required="no" select="'Psalm100.xml'"/>  <!-- the input file URI -->
    <xsl:param name="parts" required="no" select="'Cantus+Altus'"/>     <!-- names of the parts to be selected, separated by "+"-->
    <xsl:variable name="selected-parts" select="tokenize($parts, '\+')"/>    
        
    <xsl:accumulator name="selected-part-ids" as="map(xs:string, xs:boolean)" initial-value="map{}" streamable="yes">
        <xsl:accumulator-rule match="part-name/text()[$selected-parts = current()]">
            <xsl:sequence select="map:put($value, ../../@id, true())"/>
        </xsl:accumulator-rule>
    </xsl:accumulator> 
        
    <xsl:mode name="m" on-no-match="shallow-copy" streamable="yes"/>    
    
    <xsl:template name="xsl:initial-template">
        <xsl:source-document streamable="true" href="{$in}" use-accumulators="selected-part-ids">
            <xsl:apply-templates mode="m"/>
        </xsl:source-document>
    </xsl:template>
    
    <xsl:template match="part" mode="m">
        <xsl:if test="map:contains(accumulator-before('selected-part-ids'), @id)">
            <xsl:next-match/>
        </xsl:if>
    </xsl:template>
    
  
    
</xsl:stylesheet>