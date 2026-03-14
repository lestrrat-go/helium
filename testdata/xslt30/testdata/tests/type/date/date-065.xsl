<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<xsl:output encoding="iso-8859-1" />

<?spec fo#casting-to-datetimes?>
  <!-- PURPOSE: XSLT 2.0: basic test of gHorribleKludge types -->

<xsl:param name="y" select="xs:gYear('2003')"/>
<xsl:param name="yz" select="xs:gYear('2003Z')"/>
<xsl:param name="yt" select="xs:gYear('2003+01:00')"/>

<xsl:param name="ym" select="xs:gYearMonth('2003-03')"/>
<xsl:param name="ymz" select="xs:gYearMonth('2003-03Z')"/>
<xsl:param name="ymt" select="xs:gYearMonth('2003-03+01:00')"/>

<xsl:param name="m" select="xs:gMonth('--03')"/>
<xsl:param name="mz" select="xs:gMonth('--03Z')"/>
<xsl:param name="mt" select="xs:gMonth('--03+01:00')"/>

<xsl:param name="md" select="xs:gMonthDay('--12-31')"/>
<xsl:param name="mdz" select="xs:gMonthDay('--12-31Z')"/>
<xsl:param name="mdt" select="xs:gMonthDay('--12-31+01:00')"/>

<xsl:param name="d" select="xs:gDay('---31')"/>
<xsl:param name="dz" select="xs:gDay('---31+00:00')"/>
<xsl:param name="dt" select="xs:gDay('---31+01:00')"/>


  <xsl:template match="/">
     <out>
        <xsl:variable name="seq"
        select="$y, $yz, $yt, $ym, $ymz, $ymt, $m, $mz, $mt, $md, $mdz, $mdt, $d, $dz, $dt"/>
        <xsl:for-each select="$seq">
          <v><xsl:value-of select="."/></v>
        </xsl:for-each> 
     </out>     
  </xsl:template>

</xsl:stylesheet>
