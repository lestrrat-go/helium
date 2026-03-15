<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" 
xmlns:xdt="http://www.w3.org/2005/04/xpath-datatypes" exclude-result-prefixes="xs xdt">

<xsl:output encoding="iso-8859-1" />

  <?spec fo#func-subtract-dateTimes-yielding-dayTimeDuration?>
  <?spec fo#func-subtract-dates-yielding-dayTimeDuration?>
  <?spec fo#func-subtract-times?>

  <!-- PURPOSE: XSLT 2.0: test date|Time - date|Time -->

<xsl:param name="dateTime1" select="xs:dateTime('2004-02-29T00:12:00Z')"/>
<xsl:param name="dateTime2" select="xs:dateTime('2003-02-28T00:08:00-05:00')"/>
<xsl:param name="date1" select="xs:date('2004-02-29Z')"/>
<xsl:param name="date2" select="xs:date('2004-01-31Z')"/>
<xsl:param name="time1" select="xs:time('00:12:00Z')"/>
<xsl:param name="time2" select="xs:time('00:11:15.3Z')"/>


  <xsl:template match="/">
     <out>
       <a><xsl:value-of select="$dateTime2 - $dateTime1"/></a>
       <a><xsl:value-of select="$date1 - $date2"/></a>
       <a><xsl:value-of select="$time1 - $time2"/></a>  
       <a><xsl:value-of select="$time2 - $time1"/></a>                          
     </out>     
  </xsl:template>

</xsl:stylesheet>
