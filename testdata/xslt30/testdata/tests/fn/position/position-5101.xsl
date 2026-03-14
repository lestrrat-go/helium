<?xml version="1.0"?> 
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- DOCUMENT: http://www.w3.org/TR/xpath20 -->
<!-- PURPOSE:  Test of 'position()' function. -->
<!-- AUTHOR:   Michael Kay based on a bug report from Andrew Welch -->
<!-- DATE:     23 Oct 2008 -->

<xsl:template match="/">
   <out>
      <xsl:value-of select="for $x in /foolist/foo return $x/position()"/>
   </out>
  </xsl:template>
 
</xsl:stylesheet>
