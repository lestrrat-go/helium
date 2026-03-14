<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0"
   expand-text="1">
   <!-- Purpose: next-match across package boundaries. Variant of -036 in which the mode is accepted as final rather than public -->
   <!-- See also override-m-016, Saxon bug 4157, and W3C bug 30412 -->
   
   <t:use-package name="http://xslt-tests/next-match-036-lib">
      <t:accept component="mode" names="m" visibility="final"/>
      <t:override>
         <t:template match="tag" mode="m">
            <tag2><t:next-match/></tag2>
         </t:template>
      </t:override>
   </t:use-package>
      
   

</t:transform>
