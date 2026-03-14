<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- Test use of base-uri attribute to xsl:evaluate  -->

    <xsl:param name="path" select="'static-base-uri()'"/>

    <xsl:template match="/">
        <out>
            <xsl:evaluate xpath="$path" base-uri="http://www.w3.org/base"/>
        </out>
    </xsl:template>
</xsl:stylesheet>

