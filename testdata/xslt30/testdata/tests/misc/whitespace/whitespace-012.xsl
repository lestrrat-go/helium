<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with LRE that contains two text nodes with comment in between. 
  				First text node has whitespace only. Cases tested:
				- second text node starts with whitespace
				- second text node does not start with whitespace -->
   <!-- Note that the comment is removed before whitespace stripping. -->

   <t:template match="/">
      <out>
         <e>   h<!--c-->   </e>
         <e>   <!--c-->h   </e>
      </out>
   </t:template>
</t:transform>
