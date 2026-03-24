<?xml version='1.0' encoding='UTF-8' ?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <!-- test use inherit-namespaces="   yes  " (with whitespace) of xsl:copy (XSLT 3.0) -->
    <?spec xslt#shallow-copy?>
    <?same-as-1.0 no?>

    <xsl:output method="xml" encoding="utf-8"/>
    <xsl:template match="/">
        <output>
            <xsl:apply-templates mode="copy"/>
        </output>
    </xsl:template>

    <xsl:template match="*" mode="copy">
        <xsl:copy inherit-namespaces="   yes  ">
            <xsl:copy-of select="@*"/>
            <xsl:apply-templates mode="#current"/>
        </xsl:copy>
    </xsl:template>

</xsl:stylesheet>
