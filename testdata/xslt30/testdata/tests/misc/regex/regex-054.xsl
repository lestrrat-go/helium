<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with matches($input?,$pattern) on a literal string where pattern uses 
  				positional metacharacters ^ and $. Includes case where $input is an empty string 
  				or an empty sequence. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:value-of select="matches('abracadabra','bra')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('abracadabra','^a.*a$')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('abracadabra','^.*$')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('abracadabra','^.*.*.*$')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('','^$')"/>
         <t:text>, </t:text>
         <t:value-of select="matches((),'^$')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('abracadabra','^A.*A$')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('abracadabra','^bra')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('abracadabra','abr$')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('abracadabra','^$')"/>
      </out>
   </t:template>
</t:transform>
