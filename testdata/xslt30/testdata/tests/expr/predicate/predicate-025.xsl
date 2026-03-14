<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: PREDICATE18 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.2 -->
    <!-- Purpose: Test of starts-with on node name. -->

<xsl:template match="doc">
  <out>
    <!-- Should match foo, foundnode, f, fsub. -->
    <xsl:for-each select="*[starts-with(name(.),'f')]">
       <xsl:value-of select="name(.)"/>
	</xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
