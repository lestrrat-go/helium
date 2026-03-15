<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: XPath 2.0: invalid date and time. -->
  <?spec fo#casting-to-datetimes?><?error FORG0001?>

  <xsl:template match="/">
    <out><xsl:value-of select="xs:dateTime('2000-02-31T12:00:00+02:30')"/></out>
  </xsl:template>

</xsl:stylesheet>