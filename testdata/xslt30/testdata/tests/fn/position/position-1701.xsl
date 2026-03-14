<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- DOCUMENT: http://www.w3.org/TR/xpath -->
<!-- SECTION:  4.1 Node Set Functions. -->
<!-- PURPOSE:   Test of '[position()=n]' optimizations. -->
<xsl:strip-space elements="*"/>
<xsl:template match="doc">
<out>;
  <xsl:copy-of select="*[position() &lt; 4]"/>;
  <xsl:copy-of select="*[position() &lt;= 3]"/>;  
  <xsl:copy-of select="*[position() &gt; 4]"/>;  
  <xsl:copy-of select="*[position() &gt;= 5]"/>;  
  <xsl:copy-of select="*[position() &gt; 3 and position() &lt; 8]"/>;  
</out>
</xsl:template>


</xsl:stylesheet>