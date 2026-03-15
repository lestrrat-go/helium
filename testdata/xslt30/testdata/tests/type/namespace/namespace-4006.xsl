<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:ped="http://www.test.com">

    <!-- Purpose: Baseline test of xsl:element; stylesheet has namespace node. -->

    <xsl:template match="doc">
        <out>
            <xsl:element name="inner"/>
        </out>
    </xsl:template>

    <xsl:template match="text()"/>

</xsl:stylesheet>
