<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase to show that xpath-default-namespace must be in the XSLT namespace 
  				if the parent element is not in the XSLT namespace (i.e. LRE).-->

   <t:template match="/">
      <output>
         <out xpath-default-namespace="http://example.org/hal"><t:value-of select="doc/item[1]/subitem"/>
            <t:text>,</t:text>
            <t:value-of select="doc/item/attribute::attr"/>
         </out>
         <out t:xpath-default-namespace="http://example.org/hal"><t:value-of select="doc/child::item/@attr"/></out>
      </output>  
   </t:template>
</t:transform>
