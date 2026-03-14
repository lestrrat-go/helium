<xsl:stylesheet exclude-result-prefixes="xs f" version="3.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:f="http://aimtec.cz/EDI">

    <xsl:output indent="yes"/>

    <!-- message template -->
    <xsl:template match="root">
        <root>
            <format1>
                <xsl:value-of select="f:format-number(value1, '000')"/>
            </format1>
            <format2>
                <xsl:value-of select="f:format-number(value2, '0000000000')"/>
            </format2>
            <ver>
                <xsl:value-of select="system-property('xsl:product-version')"/>
            </ver>
            
           <!-- <xsl:for-each select="available-system-properties()">
                <xsl:if test="not(string(.) = ('bin:version', 'file:version', 'archive:version'))">
                    <a>
                        <xsl:value-of select="concat(., ' : ', system-property(string(.)))"/>
                    </a>
                </xsl:if>
            </xsl:for-each>-->
        </root>
    </xsl:template>

    <xsl:function name="f:format-number" as="xs:string">
        <xsl:param name="number"/>
        <xsl:param name="format" as="xs:string"/>
        <xsl:param name="decimals"/>

        <xsl:variable name="coefficient">
            <xsl:if test="string($decimals) != ''">
                <xsl:value-of select="1"/>
                <xsl:value-of select="
                        for $i in (1 to xs:integer($decimals))
                        return
                            '0'" separator=""/>
            </xsl:if>
        </xsl:variable>

        <xsl:choose>
            <xsl:when test="string($number) = ''">
                <xsl:value-of select="''"/>
            </xsl:when>
            <xsl:when test="string($decimals) != '' and $number castable as xs:decimal">
                <xsl:sequence select="format-number($number * $coefficient, $format)"/>
            </xsl:when>
            <xsl:when test="$number castable as xs:decimal">
                <xsl:value-of select="format-number($number, $format)"/>
            </xsl:when>
            <xsl:otherwise>
                <xsl:message terminate="yes" select="concat('ERROR: Value &quot;', $number, '&quot; is not a number!')"/>
            </xsl:otherwise>
        </xsl:choose>
    </xsl:function>
    <xsl:function name="f:format-number" as="xs:string">
        <xsl:param name="number"/>
        <xsl:param name="format" as="xs:string"/>

        <xsl:value-of select="f:format-number($number, $format, '')"/>
    </xsl:function>
</xsl:stylesheet>
