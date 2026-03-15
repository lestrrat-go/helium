<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" 
  xmlns:f="f.uri"
  xmlns:fn="http://www.w3.org/2005/xpath-functions" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
  exclude-result-prefixes="f fn xs math">

  <!--  test availability of functions within xsl:evaluate -->
  <!-- access to a private function should fail -->

  <xsl:param name="e1">true()</xsl:param>
  <xsl:param name="e2">concat('a', 'b')</xsl:param>
  <xsl:param name="e3">fn:position()</xsl:param>
  <xsl:param name="e4">xs:integer('3')</xsl:param>
  <xsl:param name="e5">f:square(5)</xsl:param>
  <xsl:param name="e6">math:pow(12,3)</xsl:param>

  <xsl:template name="main">
    <out>
      <a><xsl:evaluate xpath="($e1)"/></a>
      <a><xsl:evaluate xpath="($e2)"/></a>
      <a><xsl:evaluate xpath="($e3)" context-item="1"/></a>
      <a><xsl:evaluate xpath="($e4)"/></a>
      <a><xsl:evaluate xpath="($e5)"/></a>
      <a><xsl:evaluate xpath="($e6)"/></a>
    </out>

  </xsl:template>

  <xsl:function name="f:square" as="xs:integer">
    <xsl:param name="x" as="xs:integer"/>
    <xsl:sequence select="$x*$x"/>
  </xsl:function>

</xsl:stylesheet>
