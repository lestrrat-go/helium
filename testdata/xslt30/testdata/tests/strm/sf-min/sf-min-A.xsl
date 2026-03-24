<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>

    
    <!-- Test of xsl:source-document with min() -->
    
    <xsl:template name="s-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="min(./BOOKLIST/BOOKS/ITEM/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with min(), filtered with a motionless predicate -->
    
    <xsl:template name="s-004" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="min(./BOOKLIST/BOOKS/ITEM[@CAT='P']/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with min(), filtered with a positional predicate -->
    
    <xsl:template name="s-005" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="min(./BOOKLIST/BOOKS/ITEM[position() lt 4]/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with min(), filtered with a positional predicate -->
    
    <xsl:template name="s-006" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="min(./BOOKLIST/BOOKS/*[position() lt 4]/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with min(), filtered with a positional predicate -->
    
    <xsl:template name="s-007" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:sequence select="min(./BOOKLIST/BOOKS/*:ITEM[position() lt 4]/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with min(), filtered using data() to make it streamable -->
    
    <xsl:template name="s-008" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(min(./BOOKLIST/BOOKS/ITEM/PAGES/data()[. &lt; 1000][. &gt; 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with min(), filtered using text() to make it streamable -->
    
    <xsl:template name="s-009" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(min(./BOOKLIST/BOOKS/ITEM/PAGES/text()[. &lt; 1000][. &gt; 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with min(), filtered using number() to make it streamable -->
    
    <xsl:template name="s-010" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(min(./BOOKLIST/BOOKS/ITEM/PAGES/number()[. &lt; 1000][. &gt; 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with min(), selecting nothing and returning the $zero result -->
    
    <xsl:template name="s-011" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min(ITEM/PAGES)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of min() selecting both streamed nodes and literals -->
    
    <xsl:template name="s-012" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min((./BOOKLIST/BOOKS/ITEM/PAGES/number(), 31, 32))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of min() selecting both streamed nodes and literals while also filtering -->
    
    <xsl:template name="s-013" use-when="true() or $RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min((tail(./BOOKLIST/BOOKS/ITEM/PAGES)/number(), 31, 32))"/>
        </out>
      </xsl:source-document>
    </xsl:template>   
    
    <!-- Test of xsl:source-document with min of a computed value -->
    
    <xsl:template name="s-015" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="format-number(min(for $d in data(outermost(//DIMENSIONS))
                                                  return let $x := tokenize($d, '\s')!number() 
                                                  return $x[1]*$x[2]*$x[3]), '99.999')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with min of a computed value -->
    
    <xsl:template name="s-016" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="xs:integer(round(min(account/transaction/(@value*2))))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with min(), filtered using copy-of() to make it streamable -->
    
    <xsl:template name="s-017" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(min(./BOOKLIST/BOOKS/ITEM/PAGES/copy-of()[. &lt; 1000][. &gt; 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with min() applied to an attribute -->
    
    <xsl:template name="s-018" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(min(account/transaction/@value))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with min() over attribute values, filtered -->
    
    <xsl:template name="s-019" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(min(account/transaction/@value[xs:decimal(.) gt 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- Test of xsl:source-document with min() over attribute values, computed -->
    
    <xsl:template name="s-020" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(min(account/transaction/abs(@value)))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- min() of a conditional value -->
    
    <xsl:template name="s-021" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(min(account/transaction/
                (if (xs:date(@date) lt xs:date('2020-01-01')) then +@value else (@value+1))))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- min() with a loop used to compute each item -->
    
    <xsl:template name="s-022" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min(BOOKLIST/BOOKS/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document calling min() using //x/y -->
    <!-- Saxon makes this streamable by rewriting //X/Y as .//Y[parent::X] -->
    
    <xsl:template name="s-023" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min(outermost(//ITEM/DIMENSIONS)!xs:NMTOKENS(.)!xs:decimal(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): applied to dates -->
    
    <xsl:template name="s-030" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min(BOOKLIST/BOOKS/ITEM/PUB-DATE/xs:date(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): applied to strings -->
    
    <xsl:template name="s-031" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min(BOOKLIST/BOOKS/ITEM/AUTHOR/string())"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): applied to durations -->
    
    <xsl:template name="s-032" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min(BOOKLIST/BOOKS/ITEM ! (xs:date(PUB-DATE) - xs:date('1970-01-01')))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): sequence contain NaN -->
    
    <xsl:template name="s-033" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min(BOOKLIST/BOOKS/ITEM/DIMENSIONS/number())"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): sequence contains incompatible data types -->
    
    <xsl:template name="s-034" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min((BOOKLIST/BOOKS/ITEM/PRICE/number(), '100'))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): sequence contains incompatible data types, error is caught -->
    
    <xsl:template name="s-035" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try>
             <xsl:value-of select="min((BOOKLIST/BOOKS/ITEM/PRICE/number(), '100'))"/>
             <xsl:catch errors="*:FORG0006" select="'caught'"/>
           </xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): grounded operand, selects nothing -->
    
    <xsl:template name="s-040" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min(data(outermost(//NOTHING)))" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): striding operand, selects nothing -->
    
    <xsl:template name="s-041" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min(/BOOKLIST/BOOKS/MAGAZINE)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): crawling operand, selects nothing -->
    
    <xsl:template name="s-042" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min(outermost(//MAGAZINE))" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): climbing operand, selects nothing -->
    
    <xsl:template name="s-043" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min(//PRICE/../@nothing)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
        <!-- Streaming min(): collation argument present -->
    
    <xsl:template name="s-050" use-when="$RUN">
      <xsl:param name="c" select="'http://www.w3.org/2005/xpath-functions/collation/codepoint'"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min(outermost(//AUTHOR)/string(.), $c)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): collation argument present, unknown collation -->
    
    <xsl:template name="s-051" use-when="$RUN">
      <xsl:param name="c" select="'http://www.w3.org/2005/xpath-functions/collation/codepoint/unknown'"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="min(outermost(//AUTHOR)/string(.), $c)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): collation argument present, unknown collation, recovery case -->
    
    <xsl:template name="s-052" use-when="$RUN">
      <xsl:param name="c" select="'http://www.w3.org/2005/xpath-functions/collation/codepoint/unknown'"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try>
            <xsl:value-of select="min(outermost(//AUTHOR)/string(.), $c)"/>
            <xsl:catch errors="*:FOCH0002" select="'caught'"/>
          </xsl:try>  
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): collation argument obtained from streamed input -->
    
    <xsl:template name="s-053" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <out>
          <xsl:value-of select="min(('a', 'b', 'c'), special/codepointCollation)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): collation argument obtained from streamed input, unknown collation -->
    
    <xsl:template name="s-054" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <out>
          <xsl:value-of select="min(('a', 'b', 'c'), special/unknownCollation)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming min(): collation argument obtained from streamed input, unknown collation, recovery case -->
    
    <xsl:template name="s-055" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <out>
          <xsl:try>
            <xsl:value-of select="min(('a', 'b', 'c'), special/unknownCollation)"/>
            <xsl:catch errors="*:FOCH0002" select="'caught'"/>
          </xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
                                 
                                                
    
</xsl:stylesheet>