<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#casting-to-datetimes?>
 <!-- PURPOSE: test that a time of 24:00:00 is allowed -->


  <xsl:template match="/">
     <out>
      <a><xsl:value-of select='xs:dateTime("1996-04-14T24:00:00.000")'/></a> 
      <a><xsl:value-of select='xs:time("24:00:00+05:00")'/></a>
      <a><xsl:value-of select='xs:time("24:00:00Z") = xs:time("19:00:00-05:00")'/></a>   
      <a><xsl:value-of select='xs:dateTime("2001-12-31T24:00:00Z") = xs:dateTime("2002-01-01T05:00:00+05:00")'/></a>  
      <a><xsl:value-of select='year-from-dateTime(xs:dateTime("2001-12-31T24:00:00Z"))'/></a>                            
                          
     </out>     
  </xsl:template>

</xsl:stylesheet>
