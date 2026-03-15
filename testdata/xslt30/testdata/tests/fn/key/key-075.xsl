<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Variant of key236 from Norm Walsh -->
  <!-- For use with key236.xml -->

  <xsl:key name="k1" match="meti" use="string-to-codepoints(text())"/>

  <xsl:key name="k1" match="item" use="string-to-codepoints(text())"/>

  <xsl:template match="/">
    <xsl:variable name="root" select="."/>
    <out>
      <xsl:for-each select="97 to 105">
        <a code="{.}" values="{string-join(key('k1', ., $root),':')}"/>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
