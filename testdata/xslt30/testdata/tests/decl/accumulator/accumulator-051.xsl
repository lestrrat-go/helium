<xsl:stylesheet version="3.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs">
    
    <xsl:param name="streamable" static="yes" as="xs:boolean"/>

    <xsl:accumulator name="state" initial-value="()" _streamable="{$streamable}">
        <xsl:accumulator-rule match="*" phase="end"   select="($value, concat(name(),')'))"/>
        <xsl:accumulator-rule match="*" phase="start" select="($value, concat('(',name()))"/>
    </xsl:accumulator>
    
    <xsl:mode use-accumulators="state" _streamable="{$streamable}"/>
    

    <xsl:template match="/">
        <root>
            <start state="{accumulator-before('state')}"/>
            <xsl:apply-templates select="*"/>
            <end state="{accumulator-after('state')}"/>
        </root>
    </xsl:template>

    <xsl:template match="*">
        <start name="{name()}" state="{accumulator-before('state')}"/>
            <xsl:apply-templates select="*"/>
        <end name="{name()}" state="{accumulator-after('state')}"/>

    </xsl:template>

</xsl:stylesheet>
