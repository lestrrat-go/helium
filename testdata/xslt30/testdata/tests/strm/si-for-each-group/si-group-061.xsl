<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math"
    version="3.0">

    <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>

    <xsl:mode _streamable="{$STREAMABLE}"/>


    <xsl:template match="/">
        <out>
            <xsl:apply-templates/>
        </out>
    </xsl:template>

    <xsl:template match="cities">

        <xsl:for-each-group select="city" group-adjacent="@name, @country" composite="yes">
            <p>
                <xsl:value-of select="current-grouping-key(), current-group()/@pop/xs:double(.)"
                    separator=", "/>
            </p>
        </xsl:for-each-group>

    </xsl:template>

</xsl:stylesheet>
