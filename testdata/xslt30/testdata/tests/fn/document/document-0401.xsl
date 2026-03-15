<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with document($uri-sequence, $base-node) function: Mini Stress test. 
  				$uri-sequence contains a sequence of files with repeating file names, $base-node 
  				is the first node in the sequence. The result sequence should not contain duplicate nodes.-->

   <t:template match="/doc">
      <out>
         <t:perform-sort select="document(a, a[1])//body">
  	         <t:sort select="string(.)"/>
  	      </t:perform-sort>
      </out>
   </t:template>
</t:transform>
