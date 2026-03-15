<?xml version="1.0"?>

<!-- TEST lazy evaluation of expressions using values returned from a function -->

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:test="http://foobar/"
>

  <xsl:template match="/"  >
    <out>
        <xsl:variable name="v1" select="test:f(/, 'a', 'item')"/>
        <xsl:variable name="v2" select="test:f(/, 'b', 'thing')"/>
        <xsl:copy-of select="$v1"/>
        <xsl:copy-of select="$v2"/>
    </out>
  </xsl:template>
  
  <xsl:function name="test:f">
    <xsl:param name="doc" as="document-node()"/>
    <xsl:param name="p"/>
    <xsl:param name="q"/>
    <xsl:variable name="v" select="$p"/>
    <xsl:variable name="nodes" select="$doc/*/*[name()=$q]"/>
    <xsl:sequence select="$nodes[@att=$v]"/>
  </xsl:function>

</xsl:stylesheet>
