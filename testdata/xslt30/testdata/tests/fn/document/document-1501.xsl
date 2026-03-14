<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <!-- PURPOSE: Test that loading the same document twice gives the same node each time. -->
   <?spec xslt#document?>

   <xsl:template match="/">
      <out>
         <assert>answer should be 1</assert>
         <result>
            <xsl:value-of select="count(document(&quot;doc09.xml&quot;) | document(&quot;doc09.xml&quot;))"/>
         </result>
      </out>
   </xsl:template>

</xsl:stylesheet>
