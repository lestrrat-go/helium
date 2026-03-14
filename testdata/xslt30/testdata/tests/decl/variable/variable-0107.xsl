<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
    <xsl:output method="xml" version="1.0" encoding="UTF-8" indent="yes"/>

    <!-- tests parameter with default value of ()  -->
    <?same-as-1.0 no?>
    <?spec xslt#parameters?>

    <xsl:template match="/">
        <result>
            <xsl:for-each select="1 to 4">
                <xsl:call-template name="test">
                    <xsl:with-param name="case" select="xs:int(.)"/>
                </xsl:call-template>
            </xsl:for-each>
        </result>
    </xsl:template>
    <xsl:template name="test">
        <xsl:param name="case" required="yes" as="xs:int"/>
        <!-- parameter p with not-supplied implicitly specified default value of () -->
        <xsl:param name="p" required="no" as="xs:int*"/>
        <xsl:choose>
            <xsl:when test="$case=1">
                <a>
                    <xsl:value-of select="$p instance of xs:int*"/>
                </a>
                <!-- Result: true -->
            </xsl:when>
            <xsl:when test="$case=2">
                <b>
                    <xsl:value-of select="count($p)"/>
                </b>
            </xsl:when>
            <xsl:when test="$case=3">
                <c>
                    <xsl:value-of select="$p[1] instance of xs:int"/>
                </c>
                <!-- Result: false (Because it's an empty sequence)-->
            </xsl:when>
            <xsl:when test="$case=4">
                <d>
                    <xsl:value-of select="$p[1] + 1"/>
                </d>
            </xsl:when>
        </xsl:choose>
    </xsl:template>
</xsl:stylesheet>
