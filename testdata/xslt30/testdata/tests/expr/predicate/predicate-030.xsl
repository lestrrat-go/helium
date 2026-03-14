<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: PREDICATE37 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 3.4 -->
  <!-- Purpose: Show that we can limit match to non-empty elements. -->

<xsl:template match="/" >
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="a" >
  <!-- If the node 'a' has a child 'b' that has a value in it,
  then show it. Otherwise don't show it. -->

  <xsl:if test='b[not(.="")]' >( <xsl:value-of select="b" /> )</xsl:if>

  <!-- If the node 'a' has a child 'c' that has a value in it,
  then show it. Otherwise don't show it. -->

  <xsl:if test='c[not(.="")]' >( <xsl:value-of select="c" /> )</xsl:if>
</xsl:template>

</xsl:stylesheet> 