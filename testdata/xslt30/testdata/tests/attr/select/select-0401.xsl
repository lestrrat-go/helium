<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: Test of unions, returned in document order. -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="doc">
  This should come out fasolatido:
  <xsl:apply-templates select="fa"/>
  This should come out doremifasolatido:
  <xsl:apply-templates select="mi | do | fa | re"/>
  This should come out do-do-remi-mi1-fasolatido-fa--so-:
  <xsl:apply-templates select="mi[@mi1='-mi1-'] | do | fa/so/@so | fa | mi/@* | re | fa/@fa | do/@do"/>
  This should come out solatidoG#:
  <xsl:apply-templates select=".//*[@so]"/>
  This should come out relatidoABb:
  <xsl:apply-templates select="*//la | //Bflat | re"/>
  This should come out domitiACD:
  <xsl:apply-templates select="fa/../mi | Aflat/natural/la | Csharp//* | /doc/do | *//ti"/>
</xsl:template>

<xsl:template match="@*">
  <xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>