<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
default-validation="preserve">

  <!-- PURPOSE: Test data() applied to typed element nodes. -->
    <?spec xslt#annotation-for-constructed-element?> 
    <?spec fo#func-data?>

<xsl:variable name="tree">
  <e1 xsl:type="xs:ID">A001</e1>
  <e2 xsl:type="xs:string">A002</e2>
  <e3 xsl:type="xs:integer">003</e3>
  <e4 xsl:type="xs:date">2002-10-20</e4>
</xsl:variable>            

  <xsl:template match="/">
    <out>
      <id test="{data($tree/e1) instance of xs:ID}" value="{data($tree/e1)}"/>     
      <string test="{data($tree/e2) instance of xs:string}" value="{data($tree/e2)}"/> 
      <integer test="{data($tree/e3) instance of xs:integer}" value="{data($tree/e3)}"/>
      <date test="{data($tree/e4) instance of xs:date}" value="{data($tree/e4)}"/>              
    </out>
  </xsl:template>

</xsl:stylesheet>