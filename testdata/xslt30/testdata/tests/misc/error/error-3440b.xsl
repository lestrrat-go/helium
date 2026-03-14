<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  name="http://www.w3.org/xslt30-tests/error-3440a"
  declared-modes='no'>

<!--
      In the case of a template rule (that is, an xsl:template element having a match attribute) 
      appearing as a child of xsl:override, it is a static error if the list of modes in the mode attribute 
      contains #all or #unnamed, or if it contains #default and the default mode is the unnamed mode, or 
      if the mode attribute is omitted when the default mode is the unnamed mode.
-->

   <xsl:use-package name="http://www.w3.org/xslt30tests/error-base-005" package-version="1.0.0">
     <xsl:override>
     
       <xsl:template match="a" mode="#unnamed">
         <a/>
       </xsl:template>
       
     </xsl:override>
   </xsl:use-package>
   
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
</xsl:package>
