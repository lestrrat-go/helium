<?xml version="1.0"?> 
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- DOCUMENT: http://www.w3.org/TR/xpath -->
<!-- SECTION: 4.1 Node Set Functions -->
<!-- PURPOSE:  Test optimization of positional predicates on a reverse axis. -->
<xsl:template match="doc">
   <out>
      <xsl:variable name="x" select="3"/>
      <xsl:copy-of select="//b/preceding-sibling::a[position() &lt; $x]"/>
   </out>
  </xsl:template>
 
</xsl:stylesheet>
