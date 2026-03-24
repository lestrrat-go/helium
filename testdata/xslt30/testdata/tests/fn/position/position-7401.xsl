<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Purpose: Test () grouping with expanded version of // axis. This test attempts to
     give an explicit representation of one possible fallacious interpretation of
     chapter//footnote[2]. -->

<xsl:output method="xml" encoding="utf-8"/>

<xsl:template match="/">
  <out>
    <xsl:for-each select="(child::chapter/descendant-or-self::node())/footnote[2]">
      <greeting>
        <xsl:value-of select="."/>
      </greeting>
      <xsl:text>
</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>


