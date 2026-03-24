<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>

    
    <!-- Test of xsl:stream with "&lt;=" -->
    
    <xsl:template name="s-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="6.58 &lt;= (./BOOKLIST/BOOKS/ITEM/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:stream with "&lt;=", filtered with a motionless predicate -->
    
    <xsl:template name="s-004" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="6.58 &lt;= (./BOOKLIST/BOOKS/ITEM[@CAT='P']/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:stream with "&lt;=", filtered with a positional predicate -->
    
    <xsl:template name="s-005" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="6.58 &lt;= (./BOOKLIST/BOOKS/ITEM[position() lt 4]/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:stream with "&lt;=", filtered with a positional predicate -->
    
    <xsl:template name="s-006" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="6.58 &lt;= (./BOOKLIST/BOOKS/*[position() lt 4]/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:stream with "&lt;=", filtered with a positional predicate -->
    
    <xsl:template name="s-007" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:sequence select="6.58 &lt;= (./BOOKLIST/BOOKS/*:ITEM[position() lt 4]/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:stream with "&lt;=", filtered using data() to make it streamable -->
    
    <xsl:template name="s-008" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="346 &lt;= (./BOOKLIST/BOOKS/ITEM/PAGES/data()[. &lt; 1000][. &gt; 0])"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:stream with "&lt;=", filtered using text() to make it streamable -->
    
    <xsl:template name="s-009" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="346 &lt;= (./BOOKLIST/BOOKS/ITEM/PAGES/text()[. &lt; 1000][. &gt; 0])"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:stream with "&lt;=", filtered using number() to make it streamable -->
    
    <xsl:template name="s-010" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="346 &lt;= (./BOOKLIST/BOOKS/ITEM/PAGES/number()[. &lt; 1000][. &gt; 0])"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:stream with "&lt;=", selecting nothing and returning false -->
    
    <xsl:template name="s-011" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="23 &lt;= (ITEM/PAGES)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of "&lt;=" selecting both streamed nodes and literals -->
    
    <xsl:template name="s-012" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="346 &lt;= ((./BOOKLIST/BOOKS/ITEM/PAGES/number(), 31, 32))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of "&lt;=" selecting both streamed nodes and literals while also filtering -->
    
    <xsl:template name="s-013" use-when="true() or $RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="31 &lt;= ((tail(./BOOKLIST/BOOKS/ITEM/PAGES)/number(), 31, 32))"/>
        </out>
      </xsl:source-document>
    </xsl:template>   
    
    <!-- Test of xsl:stream with "&lt;=" of a computed value -->
    
    <xsl:template name="s-015" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="22.5 &lt;= (for $d in data(outermost(//DIMENSIONS)) return tokenize($d, '\s')!number())"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:stream with "&lt;=" of a computed value -->
    
    <xsl:template name="s-016" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="8.64 &lt;= (account/transaction/(@value*2))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:stream with "&lt;=", filtered using copy-of() to make it streamable -->
    
    <xsl:template name="s-017" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="346 &lt;= (./BOOKLIST/BOOKS/ITEM/PAGES/copy-of()[. &lt; 1000][. &gt; 0])"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:stream with "&lt;=" applied to an attribute -->
    
    <xsl:template name="s-018" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="4.32 &lt;= (account/transaction/@value)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:stream with "&lt;=" over attribute values, filtered -->
    
    <xsl:template name="s-019" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="4.32 &lt;= (account/transaction/@value[xs:decimal(.) gt 0])"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- Test of xsl:stream with "&lt;=" over attribute values, computed -->
    
    <xsl:template name="s-020" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="4.32 &lt;= (account/transaction/abs(@value))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "&lt;=" of a conditional value -->
    
    <xsl:template name="s-021" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="4.32 &lt;= (account/transaction/
                (if (xs:date(@date) lt xs:date('2020-01-01')) then +@value else (@value+1)))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "&lt;=" with a loop used to compute each item -->
    
    <xsl:template name="s-022" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="5.0 &lt;= (BOOKLIST/BOOKS/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:stream calling "&lt;=" using //x/y -->
    <!-- Saxon makes this streamable by rewriting //X/Y as .//Y[parent::X] -->
    
    <xsl:template name="s-023" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="5.0 &lt;= (/*/*/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test where the streamed operand is a mix of untypedAtomic and xs:decimal, and
         the unstreamed operand is a mix of untypedAtomic and xs:double, True result -->
    
    <xsl:template name="s-024" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
         <xsl:variable name="extra" as="item()*">
          <e>109</e>
          <e>110</e>
          <e>111</e>
        </xsl:variable>
        <xsl:variable name="other" as="item()*">
          <e>8.00</e>
          <xsl:sequence select="13.22e0"/>
          <e>8.00</e>
          <e>5.00</e>
          <xsl:sequence select="13.22e1"/>
        </xsl:variable>  
        <out>
          <xsl:value-of select="(5.0, $other) &lt;= ($extra, /*/*/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>      
    
    <!-- Test where the streamed operand is a mix of untypedAtomic and xs:decimal, and
         the unstreamed operand is a mix of untypedAtomic and xs:double, False result -->
    
    <xsl:template name="s-025" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
         <xsl:variable name="extra" as="item()*">
          <e>109</e>
          <e>110</e>
          <e>111</e>
        </xsl:variable>
        <xsl:variable name="other" as="item()*">
          <e>8.001</e>
          <xsl:sequence select="13.22e0"/>
          <e>8.002</e>
          <e>5.001</e>
          <xsl:sequence select="13.22e1"/>
        </xsl:variable>  
        <out>
          <xsl:value-of select="(5.1, $other) &lt;= ($extra, /*/*/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>    
    
  <!-- Test where the streamed operand is a mix of untypedAtomic and xs:decimal, and
         the unstreamed operand is a mix of untypedAtomic and xs:double. Result
         may be an error ("109" cannot be convered to xs:date) or true (5.1 lt 109). -->
    
    <xsl:template name="s-026" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
         <xsl:variable name="extra" as="item()*">
          <e>109</e>
          <e>110</e>
          <e>111</e>
        </xsl:variable>
        <xsl:variable name="other" as="item()*">
          <e>8.001</e>
          <xsl:sequence select="13.22e0"/>
          <e>8.002</e>
          <e>5.001</e>
          <xsl:sequence select="13.22e1"/>
          <xsl:sequence select="current-date()"/>
        </xsl:variable>  
        <out>
          <xsl:value-of select="(5.1, $other) &lt;= ($extra, /*/*/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Compare string to decimal. Error result. -->
    
    <xsl:template name="s-027" use-when="$RUN">
      <xsl:param name="s" select="'invalid'"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="$s &lt;= (/*/*/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Compare double to decimal. Success. -->
    
    <xsl:template name="s-028" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="5.0 &lt;= (/*/*/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>                                          
    
</xsl:stylesheet>