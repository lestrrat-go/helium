<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with LRE that contains two text nodes with processing-instruction in 
  				between. First text node has whitespace only. Cases tested:
				- second text node starts with whitespace
				- second text node does not start with whitespace -->

   <t:template match="/">
      <out>
         <e>   <?pi?>h</e>
         <e>h<?pi?>   </e>
      </out>
   </t:template>
</t:transform>
