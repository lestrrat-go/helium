<xsl:stylesheet version="2.0"
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>

<!-- test case for erratum FO.E42 -->
<!-- unnamed namespace nodes are deep-equal -->
<!-- Author: Michael Kay -->
<!-- Date: 2 February 2009 -->

<xsl:template match="/">
  <xsl:variable name="n1" as="node()">
    <xsl:namespace name="n1">http://n1.com/"</xsl:namespace>
  </xsl:variable>
  <xsl:variable name="n2" as="node()">
    <xsl:namespace name="n2">http://n2.com/"</xsl:namespace>
  </xsl:variable>
  <xsl:variable name="n3" as="node()">
    <xsl:namespace name="n1">http://n1.com/"</xsl:namespace>
  </xsl:variable>
  <xsl:variable name="n4" as="node()">
    <xsl:namespace name="">http://n4.com/"</xsl:namespace>
  </xsl:variable>
  <xsl:variable name="n5" as="node()">
    <xsl:namespace name="">http://n4.com/"</xsl:namespace>
  </xsl:variable>
  <a one="{deep-equal($n1, $n2)}"
     two="{deep-equal($n1, $n3)}"
     three="{deep-equal($n3, $n4)}"
     four="{deep-equal($n4, $n5)}"
     five="{deep-equal(($n1, $n4), ($n3, $n5))}"/>
</xsl:template>


  
</xsl:stylesheet>
