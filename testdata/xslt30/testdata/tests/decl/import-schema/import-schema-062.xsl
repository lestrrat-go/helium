<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema068.uri/"
        exclude-result-prefixes="xs">

<!-- PURPOSE: Test template rules matching on content type -->
<?spec xpath#id-element-test?>
<?spec xslt#patterns?>




  <xsl:template match="/">
    <xsl:variable name="t">
      <a xsl:type="xs:integer">23</a>
      <a xsl:type="xs:date">2004-10-03</a>
      <a xsl:type="xs:byte">12</a>
      <b xsl:type="xs:byte">12</b>      
     </xsl:variable>
     
     <out>
     <xsl:apply-templates select="$t" mode="types"/>
     </out>
  </xsl:template>
  
  <xsl:template match="element(*, xs:decimal)" mode="types">
    <decimal><xsl:copy-of select="."/></decimal>
  </xsl:template>

  <xsl:template match="element(*, xs:date)" mode="types">
    <date><xsl:copy-of select="."/></date>
  </xsl:template>  
  
  <xsl:template match="*" mode="types">
    <dregs><xsl:copy-of select="."/></dregs>
  </xsl:template>   
</xsl:stylesheet>