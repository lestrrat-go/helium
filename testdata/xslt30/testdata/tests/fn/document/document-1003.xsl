<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:ped="http://ped.com/"
                exclude-result-prefixes="ped">

   <?spec xslt#document?>
    <!-- Purpose: When document('') refers to the root node of the stylesheet,
     it means the current file, not the main stylesheet. In this test, the call to
     document() is in the included stylesheet, hence local to it. -->
  <!-- Remember: every top-level node in the stylesheet must have a namespace. -->

<xsl:include href="document-1001.xsl"/>

<xsl:template match="doc">
  <o><xsl:apply-templates/></o>
</xsl:template>

<ped:test attrib="yeha">Oops</ped:test>
<ped:test attrib="haye">Wrong item from main</ped:test>

</xsl:stylesheet>