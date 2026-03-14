<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <?spec fo#casting-to-datetimes?>
 <!-- PURPOSE: test casting dates to GDay and friends; with timezones -->

  
  <xsl:param name="gd" select="xs:gDay('---15+03:00')"/>
  <xsl:param name="gmd" select="xs:gMonthDay('--02-15+03:00')"/>  
  <xsl:param name="gm" select="xs:gMonth('--02+03:00')"/>  
  <xsl:param name="gym" select="xs:gYearMonth('2001-02+03:00')"/>   
  <xsl:param name="gy" select="xs:gYear('2001+03:00')"/>        

  <xsl:template match="/">
     <out>
     <true>
      <a><xsl:value-of select="xs:gDay(xs:date('2001-02-15+03:00')) eq $gd"/></a> 
      <a><xsl:value-of select="xs:gMonthDay(xs:date('2001-02-15+03:00')) eq $gmd"/></a> 
      <a><xsl:value-of select="xs:gMonth(xs:date('2001-02-15+03:00')) eq $gm"/></a> 
      <a><xsl:value-of select="xs:gYearMonth(xs:date('2001-02-15+03:00')) eq $gym"/></a> 
      <a><xsl:value-of select="xs:gYear(xs:date('2001-02-15+03:00')) eq $gy"/></a> 
     </true>
     <false>
      <a><xsl:value-of select="xs:gDay(xs:date('2002-03-15+02:00')) eq $gd"/></a> 
      <a><xsl:value-of select="xs:gMonthDay(xs:date('2002-02-15+02:00')) eq $gmd"/></a> 
      <a><xsl:value-of select="xs:gMonth(xs:date('2002-02-15+02:00')) eq $gm"/></a> 
      <a><xsl:value-of select="xs:gYearMonth(xs:date('2002-02-15+02:00')) eq $gym"/></a> 
      <a><xsl:value-of select="xs:gYear(xs:date('2002-02-15+02:00')) eq $gy"/></a>  
     </false>     
     <false>
      <a><xsl:value-of select="xs:gDay(xs:date('2002-03-15')) eq $gd"/></a> 
      <a><xsl:value-of select="xs:gMonthDay(xs:date('2002-02-15')) eq $gmd"/></a> 
      <a><xsl:value-of select="xs:gMonth(xs:date('2002-02-15')) eq $gm"/></a> 
      <a><xsl:value-of select="xs:gYearMonth(xs:date('2002-02-15')) eq $gym"/></a> 
      <a><xsl:value-of select="xs:gYear(xs:date('2002-02-15')) eq $gy"/></a>  
     </false>                                                  
     </out>     
  </xsl:template>

</xsl:stylesheet>
