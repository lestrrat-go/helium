<?xml version="1.0" ?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#date-time-values?>
  <!-- PURPOSE: XPath 2.0: ordering relationships between dates. -->

  <xsl:template match="/">
    <out>
     <xsl:variable name="a" select="xs:date('2000-01-01+02:30')"/>
     <xsl:variable name="b" select="xs:date('2000-01-01+02:30')"/>
     <xsl:variable name="c" select="xs:date('2000-01-02-05:00')"/> 
     <xsl:variable name="d" select="xs:date('2000-01-02Z')"/>                      
     <true><xsl:value-of select="$a &lt;= $b"/></true>
     <false><xsl:value-of select="$a &gt; $b"/></false>
     <true><xsl:value-of select="$a &lt; $c"/></true>
     <false><xsl:value-of select="$a &gt; $c"/></false>
     <true><xsl:value-of select="$c &gt; $d"/></true>     
     <false><xsl:value-of select="$c &lt; $d"/></false>     
     
    </out>     
  </xsl:template>

</xsl:stylesheet>