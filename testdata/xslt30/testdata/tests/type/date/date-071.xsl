<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" 
xmlns:xdt="http://www.w3.org/2005/04/xpath-datatypes" exclude-result-prefixes="xs xdt">

<xsl:output encoding="iso-8859-1" />

<?spec fo#func-add-dayTimeDuration-to-time?>

  <!-- PURPOSE: XSLT 2.0: test time +/- duration -->

<xsl:param name="date" select="xs:time('00:12:00Z')"/>
<xsl:param name="d5" select="xs:dayTimeDuration('P5DT4H0M0.3S')"/>
<xsl:param name="d-5" select="xs:dayTimeDuration('-P5DT4H0M0.3S')"/>

  <xsl:template match="/">
     <out>
       <a><xsl:value-of select="$date + $d5"/></a>
       <a><xsl:value-of select="$date + $d-5"/></a>
                                         
       <b><xsl:value-of select="$date - $d5"/></b>
       <b><xsl:value-of select="$date - $d-5"/></b>

       <a><xsl:value-of select="$d5 + $date"/></a>
       <a><xsl:value-of select="$d-5 + $date"/></a>         
     </out>     
  </xsl:template>

</xsl:stylesheet>
