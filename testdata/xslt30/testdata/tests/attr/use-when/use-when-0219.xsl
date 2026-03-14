<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test case that uses use-when=true() on the on the xsl:include but the 
  				included stylesheet has a use-when=false().  -->

   <t:include href="include2.xsl" use-when="true()"/>

   <t:template match="doc">
      <out>
         <t:apply-templates/>
      </out>
   </t:template>

   <t:template match="a" use-when="true()">
      <print_a>
         <t:next-match/>
      </print_a>
   </t:template>
   
</t:transform>
