<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
default-validation="preserve">

  <!-- PURPOSE: Test data() applied to typed attribute nodes. -->
    <?spec xslt#annotation-for-constructed-attribute?> 
    <?spec fo#func-data?>  

<xsl:variable name="tree">
  <e1><xsl:attribute name="id" type="xs:ID">A001</xsl:attribute></e1>
  <e2><xsl:attribute name="id" type="xs:string">A002</xsl:attribute></e2>
  <e3><xsl:attribute name="id" type="xs:integer">003</xsl:attribute></e3>
  <e4><xsl:attribute name="id" type="xs:date">2002-10-20</xsl:attribute></e4>
</xsl:variable>            

  <xsl:template match="/">
    <out>
      <id test="{data($tree/e1/@id) instance of xs:ID}" value="{data($tree/e1/@id)}"/>     
      <string test="{data($tree/e2/@id) instance of xs:string}" value="{data($tree/e2/@id)}"/> 
      <integer test="{data($tree/e3/@id) instance of xs:integer}" value="{data($tree/e3/@id)}"/>
      <date test="{data($tree/e4/@id) instance of xs:date}" value="{data($tree/e4/@id)}"/>              
    </out>
  </xsl:template>

</xsl:stylesheet>