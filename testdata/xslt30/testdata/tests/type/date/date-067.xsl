<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<xsl:output encoding="iso-8859-1" />

  <?spec fo#casting-to-datetimes?>
  <!-- PURPOSE: XSLT 2.0: test dateTime to gHorribleKludge conversion -->

<xsl:param name="d" select="xs:dateTime('2000-02-29T12:35:35+02:00')"/>


  <xsl:template match="/">
     <out>
        <v><xsl:value-of select="xs:gYear($d)"/></v>
        <v><xsl:value-of select="xs:gYearMonth($d)"/></v>
        <v><xsl:value-of select="xs:gMonth($d)"/></v>
        <v><xsl:value-of select="xs:gMonthDay($d)"/></v>
        <v><xsl:value-of select="xs:gDay($d)"/></v>                                
     </out>     
  </xsl:template>

</xsl:stylesheet>
