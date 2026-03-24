<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
default-validation="preserve">

  <!-- PURPOSE: Test xsl:type on literal result elements. -->
  <?spec xslt#validation-xsl-type?>
  <?spec xslt#setting-annotation-for-lre?>

<xsl:variable name="tree">
  <e1 xsl:type="xs:ID">A001</e1>
  <e2 xsl:type="xs:string">A002</e2>
  <e3 xsl:type="xs:integer">003</e3>
  <e4 xsl:type="xs:date">2002-10-20</e4>
</xsl:variable>            

  <xsl:template match="/">
    <out>
      <string><xsl:value-of select="$tree/*[. instance of element(*, xs:string)]" separator=" "/></string> 
      <integer><xsl:value-of select="$tree/*[. instance of element(*, xs:integer)]" separator=" "/></integer>         
      <date><xsl:value-of select="$tree/*[. instance of element(*, xs:date)]" separator=" "/></date>               
    </out>
  </xsl:template>

</xsl:stylesheet>