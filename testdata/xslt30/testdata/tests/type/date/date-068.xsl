<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<xsl:output encoding="iso-8859-1" />

<?spec fo#func-gDay-equal?>
<?spec fo#func-gMonthDay-equal?>
<?spec fo#func-gMonth-equal?>
<?spec fo#func-gYearMonth-equal?>
<?spec fo#func-gYear-equal?>

  <!-- PURPOSE: XSLT 2.0: test equality matching on gHorribleKludge types -->
  <!-- for reproducibility, uses timezones that are unlikely to match the implicit timezone -->

<xsl:param name="seq" select="
xs:gYear('2003'),
xs:gYear('2003-03:15'),
xs:gYear('2003+01:33'),

xs:gYearMonth('2003-03'),
xs:gYearMonth('2003-03-03:15'),
xs:gYearMonth('2003-03+01:33'),

xs:gMonth('--03'),
xs:gMonth('--03-03:15'),
xs:gMonth('--03+01:33'),

xs:gMonthDay('--12-31'),
xs:gMonthDay('--12-31-03:15'),
xs:gMonthDay('--12-31+01:33'),

xs:gDay('---31'),
xs:gDay('---31-03:15'),
xs:gDay('---31+01:33'),

xs:gYear('2004'),
xs:gYear('2004-03:15'),
xs:gYear('2004+01:33'),

xs:gYearMonth('2004-03'),
xs:gYearMonth('2004-03-03:15'),
xs:gYearMonth('2004-03+01:33'),

xs:gMonth('--02'),
xs:gMonth('--02-03:15'),
xs:gMonth('--02+01:33'),

xs:gMonthDay('--12-21'),
xs:gMonthDay('--12-21-03:15'),
xs:gMonthDay('--12-21+01:33'),

xs:gDay('---21'),
xs:gDay('---21+00:09'),
xs:gDay('---21+01:33'),

xs:gYear('2003'),
xs:gYear('2003-03:15'),
xs:gYear('2003+01:33'),

xs:gYearMonth('2003-03'),
xs:gYearMonth('2003-03-03:15'),
xs:gYearMonth('2003-03+01:33'),

xs:gMonth('--03'),
xs:gMonth('--03-03:15'),
xs:gMonth('--03+01:33'),

xs:gMonthDay('--12-31'),
xs:gMonthDay('--12-31-03:15'),
xs:gMonthDay('--12-31+01:33'),

xs:gDay('---31'),
xs:gDay('---31-03:15'),
xs:gDay('---31+01:33')"/>
  <xsl:template match="/">
     <out>
        <xsl:for-each select="distinct-values($seq)">
          <v><xsl:value-of select="."/></v>
        </xsl:for-each> 
     </out>     
  </xsl:template>

</xsl:stylesheet>
