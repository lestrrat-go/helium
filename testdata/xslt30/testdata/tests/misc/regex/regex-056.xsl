<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Simple testcase with matches($input?,$pattern) on a string literal where pattern is more 
  				commonly used regular expressions from  XML Schema Part2:Datatypes second edition. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:text>
</t:text>
         <t:value-of select="matches('...and everywhere that Mary went...','Mary')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('Special characters must be escaped','.*')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('Special characters must be escaped.*','\.\*')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('Mary had a little lamb','.a')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('Mary had a little lamb','(Mary)( )(had)')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('Mary had a little lamb','[a-z]a')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('Mary had a little lamb','[^a-z]a*')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('The pet store sold cats, dogs, and birds.','cat|dog|bird')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('=first first= # =second second=','=first|second=')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('=first first= # =second second=','(=)(first)|(second)(=)')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('=first first= # =second second= # =first= # =second=','=(first|second)=')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('Match with zero in the middle: @@Subexpresion occurs, but...: @=+=ABC@Lots of occurrences: @=+==+==+==+==+=@Must repeat entire pattern: @=+==+=+==+=@','@(=+=)*@')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('AAAD','A+B*C?D')"/>
         <t:text>
</t:text>
         <t:value-of select="matches('aaaaa bbbbb ccccc','a{5} b{2,6} c{4,8}')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('aaaaa bbbbb ccccc','a+ b{3,} c?')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('aaaaa bbbbb ccccc','a+ b{3,} c?')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('jkl abc xyz','(abc|xyz) (abc|xyz)')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('this','th.*s')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('this line matches too much','th.*s')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('this line','th[^s]*.')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('this line','th.*?s')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('this line','\w')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('this line','\W')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('this line','\s')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('a','[abc]')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('d','[^abc]')"/>
         <t:text>
</t:text>
         <t:value-of select="matches('AAABBBC','A+B*C?D')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('aaa bbb ccc','a{5} b{2,6} c{4,8}')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('aaaaa bb ccccc','a+ b{3,} c?')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('aaaaa bbbbb ccccc','a{5} b{6,} c{4,8}')"/>
      </out>
   </t:template>
</t:transform>
