<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
default-validation="preserve">

  <!-- PURPOSE: Test "instance of element(*,T)". -->
  <?spec xpath#id-element-test?>

<xsl:variable name="tree">
  <e1><xsl:element name="id" type="xs:ID">A001</xsl:element></e1>
  <e2><xsl:element name="id" type="xs:string">A002</xsl:element></e2>
  <e3><xsl:element name="id" type="xs:integer">003</xsl:element></e3>
  <e4><xsl:element name="id" type="xs:date">2002-10-20</xsl:element></e4>
</xsl:variable>            

  <xsl:template match="/">
    <out>
      <string><xsl:value-of select="$tree/*[id instance of element(*,xs:string)]/id" separator=" "/></string> 
      <integer><xsl:value-of select="$tree/*[id instance of element(*,xs:integer)]/id" separator=" "/></integer>         
      <date><xsl:value-of select="$tree/*[id instance of element ( * , xs:date )? ]/id" separator=" "/></date>               
    </out>
  </xsl:template>

</xsl:stylesheet>