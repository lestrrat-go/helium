<?xml version="1.0"?>

<xsl:stylesheet  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="2.0" >

<!-- test identity of nodes in a result tree fragment -->
<?spec xslt#generate-id?>

<xsl:variable name="fredRTF" >
    <Fred/>
</xsl:variable>
<xsl:variable name="fred" select="$fredRTF/node()" />

<xsl:variable name="billRTF" >
    <Bill/>
</xsl:variable>
<xsl:variable name="bill" select="$billRTF/node()" />

<xsl:template match="/">
    <Result>
        <xsl:copy-of select="$fred | $bill" />
    </Result>
    <!--<xsl:message>Should have Fred and Bill in Result</xsl:message>-->
</xsl:template>

</xsl:stylesheet>
