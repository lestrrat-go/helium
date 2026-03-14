<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <!-- PURPOSE: Test use of a wannabe-URI containing an unescaped space. -->


 <xsl:template name="main">
  <out>
   <xsl:copy-of select="document('dir with space/mdocs01.xml')"/>
  </out>
 </xsl:template>

</xsl:stylesheet>
