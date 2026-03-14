<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test case that uses use-when on xsl:include with an EBV of false. 
  				This also verifies that although one of the included stylesheet 
  				does not exist, no error is raised when the use-when attribute 
  				is set to false.  -->

   <t:include href="include1.xsl" use-when="false()"/>

   <t:include href="false_include.xsl" use-when="false()"/>

   <t:template match="elem">
      <out>
         <t:apply-templates select="a"/>
      </out>
   </t:template>

   <t:template match="a" use-when="true()">
      <print_a>
         <t:next-match/>
      </print_a>
   </t:template>
</t:transform>
