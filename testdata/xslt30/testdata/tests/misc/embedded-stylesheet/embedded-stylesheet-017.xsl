<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case in which a standalone standard stylesheet includes a standalone simplified stylesheet. -->

   <t:include href="standalone_simplified.xsl"/>

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
	     <out>
         <t:apply-templates/>
      </out>
	  </t:template>
</t:transform>
