<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <?spec fo#date-time-values?>
 <!-- PURPOSE: test handling of midnight -->
  
  <xsl:template match="/">
     <out>
     <time>
      <a><xsl:value-of select="xs:time('24:00:00')"/></a> 
      <a><xsl:value-of select="xs:time('24:00:00') - xs:time('23:59:59')"/></a> 
      <a><xsl:value-of select="xs:time('24:00:00') gt xs:time('23:59:59')"/></a> 
      <a><xsl:value-of select="hours-from-time(xs:time('24:00:00'))"/></a> 
      <a><xsl:value-of select="dateTime(xs:date('1999-12-31'), xs:time('24:00:00'))"/></a> 
      <a><xsl:value-of select="max((xs:time('24:00:00'), xs:time('23:59:59')))"/></a> 
     </time>
     <dateTime>
      <a><xsl:value-of select="xs:dateTime('1999-12-31T24:00:00')"/></a> 
      <a><xsl:value-of select="xs:dateTime('1999-12-31T24:00:00') - xs:dateTime('1999-12-31T23:59:59')"/></a> 
      <a><xsl:value-of select="xs:dateTime('1999-12-31T24:00:00') gt xs:dateTime('1999-12-31T23:59:59')"/></a> 
      <a><xsl:value-of select="year-from-dateTime(xs:dateTime('1999-12-31T24:00:00'))"/></a> 
      <a><xsl:value-of select="hours-from-dateTime(xs:dateTime('1999-12-31T24:00:00'))"/></a>  
      <a><xsl:value-of select="xs:time(xs:dateTime('1999-12-31T24:00:00'))"/></a> 
      <a><xsl:value-of select="xs:date(xs:dateTime('1999-12-31T24:00:00'))"/></a>             
     </dateTime>                                                
     </out>     
  </xsl:template>

</xsl:stylesheet>
