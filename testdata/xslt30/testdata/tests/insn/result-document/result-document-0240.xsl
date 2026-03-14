<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://example1.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             exclude-result-prefixes="my">
<!-- Purpose: Test the union of @cdata-section-elements values from xsl:result-document and one xsl:output declaration. -->

   <t:output name="my:temp-output"
             method="xml"
             encoding="UTF-8"
             indent="no"
             version="1.0"
             cdata-section-elements="item2 item5 item7"/>

   <t:template match="doc">
      <t:result-document format="my:temp-output" cdata-section-elements="item2 item3">
         <out>
            <item1>this is item1 - not in a cdata section</item1>
            <item2>this is item2 - in a cdata section</item2>
            <item3>this is item3 - in a cdata section</item3>
            <item4>this is item3 - not in a cdata section</item4>
            <item5>this is item3 - in a cdata section</item5>
            <item6>this is item3 - not in a cdata section</item6>
            <item7>this is item3 - in a cdata section</item7>
         </out>
      </t:result-document>
   </t:template>
</t:transform>
