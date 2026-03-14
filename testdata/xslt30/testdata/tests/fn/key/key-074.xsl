<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- test key() function with three arguments -->

  <xsl:key name="k1" match="meti" use="for $v in 1, $i in string-to-codepoints(.) return $i"/>

  <xsl:key name="k1" match="item">
    <xsl:variable name="i" select="1"/>
    <xsl:for-each select="string-to-codepoints(.)">
      <xsl:sort select="."/>
      <xsl:variable name="n" select="$i + . -$i"/>
      <xsl:sequence select="$n"/>
    </xsl:for-each>
  </xsl:key>


  <xsl:template match="/">
    <xsl:variable name="root" select="*"/>
    <out>
      <xsl:for-each select="97 to 105">
        <a code="{.}" values="{string-join(key('k1', ., $root),':')}"/>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
