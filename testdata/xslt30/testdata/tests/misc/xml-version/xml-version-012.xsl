<?xml version="1.1" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests a character that was not permitted in XML 1.0 (#x0346 character) 
   in the QNames for LRE when the XML version of the stylesheet is 1.1.  -->

   <t:output method="xml" version="1.1" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <pre:test͆ xmlns:pre="http://www.example.com/pre͆"/>
      </out>
  </t:template>
</t:transform>
