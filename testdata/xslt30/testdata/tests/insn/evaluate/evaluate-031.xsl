<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- Test use of XPath 3.0 constructs (and local variables) within xsl:evaluate  -->

    <xsl:param name="path" select="'let $x := 3 return $x*$x'"/>

    <xsl:template match="/">
        <out>
            <xsl:evaluate xpath="$path"/>
        </out>
    </xsl:template>
</xsl:stylesheet>

