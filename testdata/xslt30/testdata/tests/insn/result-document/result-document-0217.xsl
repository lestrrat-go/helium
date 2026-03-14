<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.mytest.example.org"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test with xsl:result-document and a whitespace 
  				separated list of QNames in @cdata-section-elements.-->

   <t:template match="/">
      <t:result-document method="xml" cdata-section-elements="item1 my:item3 item5">
         <out>
            <item1>a &amp; b</item1>
            <item2>a &amp; b</item2>
            <item3>a &amp; b</item3>
            <item3 xmlns="http://www.mytest.example.org">a &amp; b</item3>
            <item4>a &amp; b</item4>
            <item5>a &amp; b</item5>
         </out>
      </t:result-document>
   </t:template>
</t:transform>
