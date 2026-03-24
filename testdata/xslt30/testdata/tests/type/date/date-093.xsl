<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" 
xmlns:xdt="http://www.w3.org/2005/04/xpath-datatypes"
 exclude-result-prefixes="xs xdt">

  <?spec fo#func-dateTime?>
  <!-- PURPOSE: XSLT 2.0: test dateTime() constructor: date and time in different timezone -->

    <xsl:param name="d1" as="xs:date" select="xs:date('2004-09-05')"/>
    <xsl:param name="d2" as="xs:date" select="xs:date('2004-09-05+02:00')"/>
    <xsl:param name="t1" as="xs:time" select="xs:time('12:15:00.432')"/>
    <xsl:param name="t2" as="xs:time" select="xs:time('12:15:00.432+03:00')"/>

  <xsl:template match="/">
     <out>
      <a><xsl:value-of select="dateTime($d1, $t1)"/></a>
      <a><xsl:value-of select="dateTime($d2, $t1)"/></a>
      <a><xsl:value-of select="dateTime($d1, $t2)"/></a>
      <a><xsl:value-of select="dateTime($d2, $t2)"/></a>
     </out>
  </xsl:template>

</xsl:stylesheet>
