<?xml version='1.0'?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <!-- test that overriding a global variable works -->

    <xsl:import href="include-0105a.xsl"/>
    <xsl:variable name="test" select="'OK'"/>

    <xsl:template match="/">
        <out>
            <one>
                <xsl:value-of select="$test"/>
            </one>
            <xsl:call-template name="two"/>
        </out>
    </xsl:template>

</xsl:stylesheet>
