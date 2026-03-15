<?xml version="1.0"?> 
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE: Test xsl:perform-sort  -->

<xsl:template match="/">
   <out>
    <xsl:variable name="x" as="item()*">
      <xsl:perform-sort select="//num">
        <xsl:sort select="number(.)"/>
      </xsl:perform-sort>
    </xsl:variable>
    <xsl:value-of select="$x" separator=","/>
   </out>
</xsl:template>
</xsl:stylesheet>
