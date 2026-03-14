<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#date-time-values?>
  <!-- PURPOSE: XPath 2.0: equality and inequality between dates. -->

  <xsl:template match="/">
    <out>
     <xsl:variable name="a" select="xs:date('2000-01-01+02:30')"/>
     <xsl:variable name="b" select="xs:date('2000-01-01+02:30')"/>
     <xsl:variable name="c" select="xs:date('2000-01-02+02:30')"/> 
     <xsl:variable name="d" select="xs:date('2000-01-02+03:30')"/>                      
     <true><xsl:value-of select="$a = $b"/></true>
     <false><xsl:value-of select="$a != $b"/></false>
     <true><xsl:value-of select="$a != $c"/></true>
     <false><xsl:value-of select="$a = $c"/></false>
     <true><xsl:value-of select="$a != $d"/></true>
     <false><xsl:value-of select="$a = $d"/></false>     
     <true><xsl:value-of select="$c != $d"/></true>     
     <false><xsl:value-of select="$c = $d"/></false>     
     
    </out>     
  </xsl:template>

</xsl:stylesheet>