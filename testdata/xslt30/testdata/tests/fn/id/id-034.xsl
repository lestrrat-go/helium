<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Try multiple child steps after id() in match patterns -->
  
<xsl:output method="xml" encoding="UTF-8" indent="no"/>

  <xsl:template match='/'>
    <out>
      <xsl:apply-templates/><!-- rely on some built-ins -->
    </out>
  </xsl:template>

  <xsl:template match='id("abc")/l1/v'>
    <match-l1-v>
      <xsl:value-of select='.'/>
    </match-l1-v>
  </xsl:template>

  <xsl:template match='id("abc")/l2/v'>
    <match-l2-v>
      <xsl:value-of select='.'/>
    </match-l2-v>
  </xsl:template>

  <xsl:template match='id("abc")/l1/l2/w'>
    <match-l1-l2-w>
      <xsl:value-of select='.'/>
    </match-l1-l2-w>
  </xsl:template>

  <xsl:template match='id("abc")/l3/v'>
    <match-l3-v>
      <xsl:value-of select='.'/>
    </match-l3-v>
  </xsl:template>

  <xsl:template match='v'>
    <match-v>
      <xsl:value-of select='.'/>
    </match-v>
  </xsl:template>

  <xsl:template match='text()'/><!-- squelch direct replay of text -->

</xsl:stylesheet>
