<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://example.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">
<!-- Purpose: The unnamed output definition is used when 
                an xsl:result-document element omits the format attribute. -->

   <t:output method="text" encoding="UTF-8"/>

   <t:output name="my:format2" method="xml" encoding="UTF-8"/>

   <t:template match="doc">
      <t:result-document>
         <out>This document should be serialized as text, not as xml because
the un-named xsl:format (i.e. no 'name' attribute) has method "text".</out>
      </t:result-document>
   </t:template>
</t:transform>
