<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
  <!-- Purpose: Use multiple levels of child axis in match patterns. Spell out 'child::' sometimes. -->
  
<xsl:output method="xml" encoding="UTF-8" indent="no"/>

  <xsl:template match="/">
    <out>
      <xsl:apply-templates/><!-- rely on some built-ins -->
    </out>
  </xsl:template>

  <xsl:template match="doc/l1/v2">
    <xsl:text>
</xsl:text>
    <match>
      <xsl:text>Rule doc/l1/v2; value of matched node:  </xsl:text>
      <xsl:value-of select='.'/>
    </match>
  </xsl:template>

  <xsl:template match="doc/l1//v3">
    <xsl:text>
</xsl:text>
    <match>
      <xsl:text>Rule doc/l1//v3; value of matched node:  </xsl:text>
      <xsl:value-of select='.'/>
    </match>
  </xsl:template>

  <xsl:template match="doc//l2/w3">
    <xsl:text>
</xsl:text>
    <match>
      <xsl:text>Rule doc//l2/w3; value of matched node:  </xsl:text>
      <xsl:value-of select='.'/>
    </match>
  </xsl:template>

  <xsl:template match="doc//l2//v4">
    <xsl:text>
</xsl:text>
    <match>
      <xsl:text>Rule doc//l2//v4; value of matched node:  </xsl:text>
      <xsl:value-of select='.'/>
    </match>
  </xsl:template>

  <xsl:template match="doc/child::l1/child::x2">
    <xsl:text>
</xsl:text>
    <match>
      <xsl:text>Rule doc/child::l1/child::x2; value of matched node:  </xsl:text>
      <xsl:value-of select='.'/>
    </match>
  </xsl:template>

  <xsl:template match="doc/child::l1//child::x3">
    <xsl:text>
</xsl:text>
    <match>
      <xsl:text>Rule doc/child::l1//child::x3; value of matched node:  </xsl:text>
      <xsl:value-of select='.'/>
    </match>
  </xsl:template>

  <xsl:template match="doc//child::l2/child::y3">
    <xsl:text>
</xsl:text>
    <match>
      <xsl:text>Rule doc//child::l2/child::y3; value of matched node:  </xsl:text>
      <xsl:value-of select='.'/>
    </match>
  </xsl:template>

  <xsl:template match="doc//child::l2//child::x4">
    <xsl:text>
</xsl:text>
    <match>
      <xsl:text>Rule doc//child::l2//child::x4; value of matched node:  </xsl:text>
      <xsl:value-of select='.'/>
    </match>
  </xsl:template>

  <xsl:template match='text()'/><!-- squelch direct replay of text -->

</xsl:stylesheet>