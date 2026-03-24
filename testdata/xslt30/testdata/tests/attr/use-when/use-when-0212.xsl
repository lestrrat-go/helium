<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test case that uses system-property function as the expression used to evaluate the
  				EBV of the use-when attribute.  -->

   <t:template match="*">
      <out>
         <t:copy>
            <t:apply-templates/>
         </t:copy>
      </out>
  </t:template>

   <t:template match="a" use-when="system-property('t:version') = '1.0'">
      <print>
         <t:next-match/>
      </print>
  </t:template>

   <t:template match="b" use-when="number(system-property('t:version')) ge 2.0">
      <print>
         <t:next-match/>
      </print>
  </t:template>
</t:transform>
