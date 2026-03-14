<?xml version="1.0"?> 
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE: A test for xsl:sort: sort by reverse document order -->

<xsl:template match="doc">
    <out>
      <xsl:for-each select="num">
        <xsl:sort data-type="number" select="position()" order="descending"/>
        <o><xsl:value-of select="."/></o>;
      </xsl:for-each>
    </out>
</xsl:template>
</xsl:stylesheet>
