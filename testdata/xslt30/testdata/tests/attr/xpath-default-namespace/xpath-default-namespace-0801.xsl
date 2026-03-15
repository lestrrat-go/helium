<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase to verify that an error is raised if the xpath-default-namespace 
  				attribute is in the XSLT namespace and its parent element is in the XSLT 
  				namespace.-->

   <t:template match="/">
      <out>
         <t:value-of select="doc/child::item/@attr"
                     t:xpath-default-namespace="http://example.org/hal"/>
      </out>  
   </t:template>
</t:transform>
