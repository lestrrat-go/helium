<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with xpath-default-namespace and unprefixed element names in the
  				 'elements' attribute of xsl:strip-space. -->

   <t:strip-space elements="doc item subitem"
                  xpath-default-namespace="http://example.org/hal"/>

   <t:preserve-space elements="doc item subitem"/>

   <t:template match="/">
      <t:text>
</t:text>
      <out>
         <t:apply-templates/>
      </out>
   </t:template>
</t:transform>
